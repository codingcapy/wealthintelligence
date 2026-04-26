import {
  useInfiniteQuery,
  useMutation,
  useQueryClient,
} from "@tanstack/react-query";
import { api, authHeaders } from "./client";
import { getSession } from "./plans";
import type {
  Generation,
  DeserializedGeneration,
  GenerationsPage,
  CreateGenerationArgs,
  DeleteGenerationArgs,
} from "./types";

export function mapSerializedGenerationToSchema(
  serialized: Generation,
): DeserializedGeneration {
  return {
    ...serialized,
    createdAt: new Date(serialized.createdAt),
  };
}

async function createGeneration(args: CreateGenerationArgs) {
  const token = getSession();
  const res = await api.post<{ generation: Generation }>("/ai/generate", args, {
    headers: authHeaders(token),
  });
  return res.data;
}

export const useCreateGenerationMutation = (
  onError?: (message: string) => void,
) => {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: createGeneration,
    onSettled: (data) => {
      queryClient.invalidateQueries({
        queryKey: ["generations", data?.generation.planId],
      });
    },
    onError: (error) => {
      if (onError) {
        onError(error.message);
      }
    },
  });
};

async function getGenerationsByPlanId(
  planId: number,
  cursor: number | null,
): Promise<GenerationsPage> {
  const token = getSession();
  const params = new URLSearchParams();
  if (cursor) params.set("cursor", String(cursor));
  const res = await api.get<{
    generations: Generation[];
    nextCursor: number | null;
  }>(`/generations/${planId}?${params.toString()}`, {
    headers: authHeaders(token),
  });
  return {
    generations: res.data.generations.map(mapSerializedGenerationToSchema),
    nextCursor: res.data.nextCursor,
  };
}

export const useInfiniteGenerationsByPlanId = (planId: number) =>
  useInfiniteQuery({
    queryKey: ["generations", planId],
    queryFn: ({ pageParam }) => getGenerationsByPlanId(planId, pageParam),
    initialPageParam: null as number | null,
    getNextPageParam: (lastPage) => lastPage.nextCursor ?? undefined,
    enabled: !!planId,
  });

async function deleteGeneration(args: DeleteGenerationArgs) {
  const token = getSession();
  const res = await api.post<{ generation: Generation }>(
    "/generations/delete",
    args,
    {
      headers: authHeaders(token),
    },
  );
  return res.data;
}

export const useDeleteGenerationMutation = (
  onError?: (message: string) => void,
) => {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: deleteGeneration,
    onSettled: (data) => {
      queryClient.invalidateQueries({
        queryKey: ["generations", data?.generation.planId],
      });
    },
    onError: (error) => {
      if (onError) {
        onError(error.message);
      }
    },
  });
};
